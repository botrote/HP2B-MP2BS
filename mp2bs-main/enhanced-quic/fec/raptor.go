package fec

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <math.h>
#include <sys/time.h>
#include <time.h>
#include <unistd.h>
#include <immintrin.h>	// AVX
#include "raptor.h"


char** MallocCharTwoArray(int row, int col)
{
	int i, j;
	char **arr = (char**)malloc(sizeof(char*)*row);

	for(i = 0; i < row; i++)
	{
		arr[i] = (char*) malloc(sizeof(char)*col);
		for (j = 0; j < col; j++)
			arr[i][j] = 0;
	}

	return arr;
}


void FreeCharTwoArray(char** arr, int row)
{
	int i;
	for(i=0; i<row; i++)
		free(arr[i]);

	free(arr);
}


int** MallocIntTwoArray(int row, int col)
{
	int i, j;
	int **arr = (int**)malloc(sizeof(int*)*row);

	for (i = 0; i < row; i++)
	{
		arr[i] = (int*) malloc(sizeof(int)*col);
		for (j = 0; j < col; j++)
			arr[i][j] = 0;
	}

	return arr;

}


void FreeIntTwoArray(int** arr, int row)
{
	int i;
	for (i = 0; i < row; i++)
		free(arr[i]);

	free(arr);
}


int* bec(int overhead, int errnum)
{
	int i, idx;
	int target = 0;
	int m[overhead];
	struct timeval t;

	for (i = 0; i < overhead; i++)
		m[i] = 0;

	gettimeofday(&t, NULL);
	srand((unsigned int)time(NULL) + (unsigned int)(t.tv_usec*000000));

	while(target != errnum)
	{
		idx = rand()%overhead;
		if(m[idx] == 1)
			continue;
		m[idx] = 1;
		target++;
	}

	int *r = (int*)malloc(sizeof(int)*(overhead-errnum));
	int cnt = 0;

	for (i = 0; i < overhead; i++)
	{
		if (m[i] != 1)
		{
			r[cnt] = i;
			cnt++;
		}
	}

	return r;
}


void RaptorInitWriteFile(char filename[], int K, int SYMBOL_SIZE, double minCR, double CR)
{
	int i, j;
	int res;
	char str[100];
	raptor_params_t *params = RaptorInitialization(K, SYMBOL_SIZE, minCR, CR);

	int fd = open(filename, O_WRONLY|O_CREAT|O_TRUNC, 0644);
	if (fd == -1)
	{
		perror("File open error");
		exit(1);
	}

	sprintf(str, "%d\n", params->K);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->SYMBOL_SIZE);
	res = write(fd, str, strlen(str));
	sprintf(str, "%f\n", params->minCR);
	res = write(fd, str, strlen(str));
	sprintf(str, "%f\n", params->CR);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->S);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->H);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->L);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->Lp);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->N);
	res = write(fd, str, strlen(str));
	sprintf(str, "%d\n", params->M);
	res = write(fd, str, strlen(str));

	for (i = 0; i < 3; i++)
	{
		for (j = 0; j < params->M; j++)
		{
			sprintf(str, "%d\n", params->U[i][j]);
			res = write(fd, str, strlen(str));
		}
	}

	for (i = 0; i < params->K; i++)
	{
		for (j = 0; j < params->L; j++)
		{
			sprintf(str, "%d\n", params->A[i][j]);
			res = write(fd, str, strlen(str));
		}
	}

	for (i = 0; i < params->K; i++)
	{
		for (j = 0; j < params->K; j++)
		{
			sprintf(str, "%d\n", params->AI[i][j]);
			res = write(fd, str, strlen(str));
		}
	}

	for (i = 0; i < (params->K + params->S); i++)
	{
		sprintf(str, "%u\n", params->m[i]);
		res = write(fd, str, strlen(str));
	}

	close(fd);

	RaptorDeinitialization(params);
}


raptor_params_t* RaptorInitFromFile(char *filename)
{
	int i, j;
	int res;
	raptor_params_t *params = (raptor_params_t*)malloc(sizeof(raptor_params_t));
	int intval;
	double doubleval;
	FILE *fd = fopen(filename, "rt");

	if (fd == NULL)
	{
		perror("File open error");
		exit(1);
	}

	res = fscanf(fd, "%d", &intval);
	params->K = intval;
	res = fscanf(fd, "%d", &intval);
	params->SYMBOL_SIZE = intval;
	res = fscanf(fd, "%lf", &doubleval);
	params->minCR = doubleval;
	res = fscanf(fd, "%lf", &doubleval);
	params->CR = doubleval;
	res = fscanf(fd, "%d", &intval);
	params->S = intval;
	res = fscanf(fd, "%d", &intval);
	params->H = intval;
	res = fscanf(fd, "%d", &intval);
	params->L = intval;
	res = fscanf(fd, "%d", &intval);
	params->Lp = intval;
	res = fscanf(fd, "%d", &intval);
	params->N = intval;
	res = fscanf(fd, "%d", &intval);
	params->M = intval;

	params->U = MallocIntTwoArray(3, params->M);

	for (i = 0; i < 3; i++)
	{
		for (j = 0; j < params->M; j++)
		{
			res = fscanf(fd, "%d\n", &intval);
			params->U[i][j] = intval;
		}
	}

	params->A = MallocCharTwoArray(params->K, params->L);

	for (i = 0; i < params->K; i++)
	{
		for (j = 0; j < params->L; j++)
		{
			res = fscanf(fd, "%d\n", &intval);
			params->A[i][j] = (char)intval;
		}
	}

	params->AI = MallocCharTwoArray(params->K, params->K);

	for (i = 0; i < params->K; i++)
	{
		for (j = 0; j < params->K; j++)
		{
			res = fscanf(fd, "%d\n", &intval);
			params->AI[i][j] = (char)intval;
		}
	}

	params->m = (unsigned int*) malloc(sizeof(unsigned int)*(params->K + params->S));

	for (i = 0; i < (params->K + params->S); i++)
	{
		res = fscanf(fd, "%d\n", &intval);
		params->m[i] = (unsigned int)intval;
	}

	fclose(fd);

	return params;
}


void FindMinOverhead(char** data, double target, int trynum, raptor_params_t *params)
{
	int ok = 0;
	int failnum;
	int i, j;
	int *r;
	int *ESIs;
	char **erc;
	double errnum;
	params->N = params->K;
	char buf[256];
	char **d;
	char **e = RaptorEncoder(data, params);

	int fd = open("FindMinOverhead.txt", O_WRONLY|O_CREAT|O_APPEND, 0644);
	if (fd == -1)
	{
		perror("File open error");
		exit(1);
	}

	while(1)
	{
		failnum = 0;
		params->N++;
		errnum = params->M - params->N;

		for(i=0; i<trynum; i++)
		{
			r = bec(params->M, (int)errnum);
			ESIs = (int*)malloc(sizeof(int)*params->N);
			erc = MallocCharTwoArray(params->N, params->SYMBOL_SIZE);

			for(j=0; j<params->N; j++)
			{
				ESIs[j] = r[j];
				memcpy(erc[j], e[r[j]], params->SYMBOL_SIZE);
			}

			d = RaptorDecoder(erc, ESIs, params);

			if(d == NULL)
				failnum++;

			free(r);
			free(ESIs);
			FreeCharTwoArray(erc, params->N);
			if (d != NULL)
				FreeCharTwoArray(d, params->K);

			if (i%10 == 0)
				printf("%d: K[%d] N[%d] Fail[%d]\n", i, params->K, params->N-params->K, failnum);

			if (((double)failnum/(double)trynum) > target)
				break;
		}

		if ((i == trynum) && (((double)failnum/(double)trynum) <= target))
			break;
	}

	printf("Found it!! K[%d] N[%d] Fail[%d]\n", params->K, params->N-params->K, failnum);

	sprintf(buf, "K[%d] N[%d] Fail[%d] - Target[%f] Trynum[%d]\n", params->K, params->N-params->K, failnum, target, trynum);
	int res = write(fd, buf, strlen(buf));

	close(fd);

	FreeCharTwoArray(e, params->M);
}


int Isp(int x)
{
	int i;

	for (i = 2; i < x; i++)
		if (x % i == 0)
			return 0;

	return 1;

}

double Factor(int n)
{
	int i;
	double v = 1;

	for (i = 1; i <= n; i++)
		v = v * i;

	return v;
}


int rg(int v)
{
	static int d[] = {1, 2, 3, 4, 10, 11, 40};
	static int f[] = {10241, 491582, 712794, 831695, 948446, 1032189, 1048576};
	int j = 0;

	while (v >= f[j])
		j++;

	return d[j];
}


int rd(int X, int i, int m)
{
	double res;
	unsigned long tmp;

	tmp = (V0[(X+i)%256] ^ V1[((unsigned int)floor((double)(X/256.0)+i))%256]);
	res = tmp / m;
	res = tmp - floor(res)*(double)m;

	return (int)res;
}


int* triple(raptor_params_t *params, int i)
{
	int Q = 65521;
	int Jk, V;
	int A, B, Y;

	Jk = J[params->K];

	A = (53591+(Jk*997)) % Q;
	B = (10267*(Jk+1)) % Q;
	Y = (B+(i*A)) % Q;

	V = rd(Y, 0, 1048576);

	int *v = (int*)malloc(sizeof(int)*3);
	v[0] = rg(V);							//d
	v[1] = 1 + rd(Y, 1, params->Lp-1);		//a
	v[2] = rd(Y, 2, params->Lp);			//b

	return v;
}


void TripleCalculate(raptor_params_t *params)
{
	int i;
	int *v;
	params->U = MallocIntTwoArray(3, params->M);

	for(i=0; i<params->M; i++)
	{
		v = triple(params, i);
		params->U[0][i] = v[0];
		params->U[1][i] = v[1];
		params->U[2][i] = v[2];

		free(v);
	}
}


void FindMatrixA(raptor_params_t *params)
{
	int i, j;
	int d, a, b;
	int SS;

	params->A = MallocCharTwoArray(params->K, params->L);

	for(i=0; i<params->K; i++)
	{
		d = params->U[0][i];
		a = params->U[1][i];
		b = params->U[2][i];

		while (b >= params->L)
			b = (b+a)%params->Lp;

		params->A[i][b] ^= 1;
		SS = min(d-1, params->L-1);

		for (j=0; j<SS; j++)
		{
			b = (b+a)%params->Lp;
			while (b >= params->L)
				b = (b+a)%params->Lp;
			params->A[i][b] ^= 1;
		}
	}
}



char** FindMatrixAUsingSymbols(int ESIs[], raptor_params_t *params)
{
	int i, j;
	int d, a, b;
	int SS;

	char **Anew = MallocCharTwoArray(params->N, params->L);

	for(i=0; i<params->N; i++)
	{
		d = params->U[0][ESIs[i]];
		a = params->U[1][ESIs[i]];
		b = params->U[2][ESIs[i]];

		while (b >= params->L)
			b = (b+a) % params->Lp;

		Anew[i][b] ^= 1;
		SS = min(d-1, params->L-1);

		for (j = 0; j < SS; j++)
		{
			b = (b+a) % params->Lp;
			while (b >= params->L)
				b = (b+a) % params->Lp;
			Anew[i][b] ^= 1;
		}
	}

	return Anew;
}


void MakeGraySequence(raptor_params_t *params)
{
	int g, gTmp, g2Tmp;
	int Hp = (int)ceil(params->H/2.0);
	int K = params->K;
	int S = params->S;
	int count = 0;
	int onecount = 0;
	int i, j;

	params->m = (unsigned int*) malloc(sizeof(unsigned int)*(K + S));
	memset(params->m, 0x00, K + S);

	for(i=0; onecount < (K + S); i++)
	{
		g = i ^ (int)(floor((double)(i/2.0)));
		gTmp = g;

		count = 0;
		for (j = 0; ; j++)
		{
			g2Tmp = g % 2;
			if (g2Tmp == 1)
				count++;

			g /= 2.0;

			if (g == 0)
			{
				if (Hp == count)
				{
					params->m[onecount] = gTmp;
					onecount++;
				}
				break;
			}
		}
	}
}



void InverseHALF(char **A, int RowNum, raptor_params_t *params)
{
	int R = RowNum;
	int K = params->K;
	int S = params->S;
	int H = params->H;
	int i, j, as, aaa, aaaa, count;

	for (as = 0; as < R; as++)
	{
		for (aaaa = 0; aaaa < H; aaaa++)
		{
			if (A[as][K+S+aaaa] == 1)
			{
				for(i=0; i < K+S; i++)
				{
					count = params->m[i];

					for(j=0; ;j++)
					{
						aaa = count % 2;
						if ((aaa == 1) && (j == aaaa))
							A[as][i] ^= 1;
						count /= 2.0;

						if (count == 0)
							break;
					}
				}
			}
		}
	}
}


void InverseLDPC(char **A, int RowNum, raptor_params_t *params)
{
	int R = RowNum;
	int K = params->K;
	int S = params->S;
	int a, b, i, as;

	for (as = 0; as < R; as++)
	{
		for (i = 0; i < K; i++)
		{
			a = 1 + ((int)(floor((double)(i/S)))%(S-1));
			b = i % S;

			if(A[as][K+b] == 1)
				A[as][i] ^= 1;
			b = (b+a)%S;
			if(A[as][K+b] == 1)
				A[as][i] ^= 1;
			b = (b+a)%S;
			if(A[as][K+b] == 1)
				A[as][i] ^= 1;
		}
	}
}


void FindInverseMatrix(raptor_params_t *params)
{
	int i, j, as;
	int K = params->K;
	char **AA = MallocCharTwoArray(K, K);
	char **AI = MallocCharTwoArray(K, K);

	params->AI = AI;

	for (i = 0; i < K; i++)
	{
		memcpy(AA[i], params->A[i], K);
		AI[i][i] = 1;
	}

	for (j = 0; j < K; j++)
	{
		i = j;
		while (AA[i][j] == 0)
		{
			i++;
			if (i == K)
			{
				printf("Can't find inverse matrix!!!\n");
				FreeCharTwoArray(AA, K);
				FreeCharTwoArray(AI, K);
				params->AI = NULL;
				return;
			}
		}

		if (i != j)
		{
			for (as = 0; as < K; as++)
			{
				AA[j][as] ^= AA[i][as];
				AI[j][as] ^= AI[i][as];
			}
		}

		for (i = 0; i < K; i++)
		{
			if ((AA[i][j] == 1) && (i != j))
			{
				for (as = 0; as < K; as++)
				{
					AA[i][as] ^= AA[j][as];
					AI[i][as] ^= AI[j][as];
				}
			}
		}
	}

	FreeCharTwoArray(AA, K);
}


char** IntermediateKSymbols(char **A, char **e, raptor_params_t *params)
{
	int i, j, as;
	int K = params->K;
	int N = params->N;
	char *ptr;
	char **AA = A;
	char **d = MallocCharTwoArray(N, params->SYMBOL_SIZE);
	char **c;

	for (i = 0; i < N; i++)
		memcpy(d[i], e[i], params->SYMBOL_SIZE);

	for (j = 0; j < K; j++)
	{
		i = j;
		while (AA[i][j] == 0)
		{
			i++;
			if (i == N)
			{
				printf("Not Full Rank Matrix!!!\n");
				FreeCharTwoArray(d, N);
				return NULL;
			}
		}

		if(i != j)
		{
			ptr = AA[j];
			AA[j] = AA[i];
			AA[i] = ptr;

			ptr = d[j];
			d[j] = d[i];
			d[i] = ptr;
		}

		for (i = 0; i < N; i++)
		{
			if ((AA[i][j] == 1) && (i != j))
			{
				// yunmin - AVX
				if (VECTOR)
				{
					vector_xor(AA[i], AA[i], AA[j], K);
					vector_xor(d[i], d[i], d[j], params->SYMBOL_SIZE);
				}
				else
				{
					for (as=0; as < K; as++)
						AA[i][as] ^= AA[j][as];
					for (as=0; as < params->SYMBOL_SIZE; as++)
						d[i][as] ^= d[j][as];
				}

				params->XORDEC += 2;
			}
		}
	}

	c = MallocCharTwoArray(params->L, params->SYMBOL_SIZE);

	for (i = 0; i < K; i++)
		memcpy(c[i], d[i], params->SYMBOL_SIZE);

	FreeCharTwoArray(d, N);

	return c;
}


char** LTCode(char **c, int SymNum, raptor_params_t *params, int type)
{
	int i, j, jj;
	int *v;
	int d, a, b;
	int SS;
	char **e = MallocCharTwoArray(SymNum, params->SYMBOL_SIZE);

	for (i = 0; i < SymNum; i++)
	{
		d = params->U[0][i];
		a = params->U[1][i];
		b = params->U[2][i];

		while (b >= params->L)
			b = (b+a) % params->Lp;

		memcpy(e[i], c[b], params->SYMBOL_SIZE);

		SS = min(d-1, params->L-1);

		for (j = 0; j < SS; j++)
		{
			b = (b+a) % params->Lp;
			while (b >= params->L)
				b = (b+a) % params->Lp;

			// yunmin - AVX
			if (VECTOR)
			{
				vector_xor(e[i], e[i], c[b], params->SYMBOL_SIZE);
			}
			else
			{
				for (jj = 0; jj < params->SYMBOL_SIZE; jj++)
					e[i][jj] ^= c[b][jj];
			}

			if (type == 0)
				params->XORENC++;
			else
				params->XORDEC++;
		}
	}

	return e;
}


raptor_params_t* RaptorInitialization(int K, int SYMBOL_SIZE, double minCR, double CR)
{
	raptor_params_t *params = (raptor_params_t*)malloc(sizeof(raptor_params_t));

	int X = 1;
	while (X * (X-1) < 2*K)
		X = X + 1;

	int S = 1;
	while (S < ceil(0.01*K) + X)
		S = S + 1;
	while(Isp(S)==0)
		S++;

	int H = 1;
	while (Factor(H) / ((Factor(ceil(H/2.0)))* Factor((H-ceil(H/2.0)))) <K+S)
		H = H + 1;

	int L = K + S + H;
	int Lp = L;
	while(Isp(Lp)==0)
		Lp++;

	int N = round((1+ minCR)* K);
	int M = round((1+ CR)* K);

	params->K = K;
	params->SYMBOL_SIZE = SYMBOL_SIZE;
	params->minCR = minCR;
	params->CR = CR;
	params->S = S;
	params->H = H;
	params->L = L;
	params->Lp = Lp;
	params->N = N;
	params->M = M;

	TripleCalculate(params);
	FindMatrixA(params);
	MakeGraySequence(params);
	InverseHALF(params->A, params->K, params);
	InverseLDPC(params->A, params->K, params);
	FindInverseMatrix(params);

	if (params->AI == NULL)
	{
		printf("RaptorInitialization failed!!\n");
		free(params->m);
		params->m = NULL;
		FreeCharTwoArray(params->A, K);
		params->A = NULL;
		free(params);
		return NULL;
	}

	return params;
}



void RaptorDeinitialization(raptor_params_t *params)
{
	if(params->m != NULL) free(params->m);
	if(params->A != NULL) FreeCharTwoArray(params->A, params->K);
	if(params->AI != NULL) FreeCharTwoArray(params->AI, params->K);
	if(params != NULL) free(params);
}

char** RaptorEncoder(char **data, raptor_params_t *params)
{
	int K = params->K;
	int H = params->H;
	int S = params->S;
	int L = params->L;
	int Lp = params->Lp;
	int M = params->M;

	int i, j, jj, a, b;
	int count, aaa, BitPos;
	unsigned int *TmpM = (unsigned int*) malloc(sizeof(unsigned int)*(K + S));

	char **c = MallocCharTwoArray(params->L, params->SYMBOL_SIZE);

	params->XORENC = 0;

	//Intermediate Symbol for [1~K]
	for (j = 0; j < K; j++)
	{
		for (i = 0; i < K; i++)
		{
			if (params->AI[j][i] == 1)
			{
				// yunmin - AVX
				if (VECTOR)
				{
					vector_xor(c[j], c[j], data[i], params->SYMBOL_SIZE);
				}
				else
				{
					for (jj = 0; jj < params->SYMBOL_SIZE; jj++)
						c[j][jj] = c[j][jj] ^ data[i][jj];
				}

				params->XORENC++;
			}
		}
	}

	//LDPC
	for (i = 0; i < K; i++)
	{
		a = 1 + ((int)(floor((double)(i/S)))%(S-1));
		b = i % S;

		// yunmin - AVX
		if (VECTOR)
		{
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
			b = (b+a)%S;
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
			b = (b+a)%S;
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
		}
		else
		{
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
			b = (b+a)%S;
			for (j=  0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
			b = (b+a)%S;
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
		}

		params->XORENC += 3;
	}

	//HALF symbol
	memcpy(TmpM, params->m, sizeof(unsigned int)*(K + S));

	for (i = 0; i < K+S; i++)
	{
		count = TmpM[i];
		BitPos = 0;

		for (j = 0; ; j++)
		{
			aaa = TmpM[i] % 2;

			if (aaa == 1)
			{
				if (BitPos == j)
				{
					// yunmin - AVX
					if (VECTOR)
					{
						vector_xor(c[K+S+j], c[K+S+j], c[i], params->SYMBOL_SIZE);
					}
					else
					{
						for (jj = 0; jj < params->SYMBOL_SIZE; jj++)
							c[K+S+j][jj] ^= c[i][jj];
					}

					params->XORENC++;
				}
			}

			BitPos++;
			TmpM[i] /= 2.0;
			if (TmpM[i] == 0)
				break;
		}
	}

	char **cc = MallocCharTwoArray(params->L, params->SYMBOL_SIZE);
	for (i = 0; i < params->L; i++)
		memcpy(cc[i], c[i], params->SYMBOL_SIZE);

	//LT coding
	char **e = LTCode(cc, M, params, 0);

	free(TmpM);
	FreeCharTwoArray(c, params->L);
	FreeCharTwoArray(cc, params->L);

	return e;
}


char** RaptorEncoder2(char **data, raptor_params_t *params, int num_syms_per_enc_blk)
{
	int K = params->K;
	int H = params->H;
	int S = params->S;
	int L = params->L;
	int Lp = params->Lp;
	int M = num_syms_per_enc_blk;

	int i, j, jj, a, b;
	int count, aaa, BitPos;
	unsigned int *TmpM = (unsigned int*) malloc(sizeof(unsigned int)*(K + S));

	char **c = MallocCharTwoArray(params->L, params->SYMBOL_SIZE);

	params->XORENC = 0;

	//Intermediate Symbol for [1~K]
	for (j = 0; j < K; j++)
	{
		for (i = 0; i < K; i++)
		{
			if (params->AI[j][i] == 1)
			{
				// yunmin - AVX
				if (VECTOR)
				{
					vector_xor(c[j], c[j], data[i], params->SYMBOL_SIZE);
				}
				else
				{
					for (jj = 0; jj < params->SYMBOL_SIZE; jj++)
						c[j][jj] = c[j][jj] ^ data[i][jj];
				}

				params->XORENC++;
			}
		}
	}

	//LDPC
	for (i = 0; i < K; i++)
	{
		a = 1 + ((int)(floor((double)(i/S)))%(S-1));
		b = i % S;

		// yunmin - AVX
		if (VECTOR)
		{
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
			b = (b+a)%S;
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
			b = (b+a)%S;
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
		}
		else
		{
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
			b = (b+a)%S;
			for (j=  0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
			b = (b+a)%S;
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
		}

		params->XORENC += 3;
	}

	//HALF symbol
	memcpy(TmpM, params->m, sizeof(unsigned int)*(K + S));

	for (i = 0; i < K+S; i++)
	{
		count = TmpM[i];
		BitPos = 0;

		for (j = 0; ; j++)
		{
			aaa = TmpM[i] % 2;

			if (aaa == 1)
			{
				if (BitPos == j)
				{
					// yunmin - AVX
					if (VECTOR)
					{
						vector_xor(c[K+S+j], c[K+S+j], c[i], params->SYMBOL_SIZE);
					}
					else
					{
						for (jj = 0; jj < params->SYMBOL_SIZE; jj++)
							c[K+S+j][jj] ^= c[i][jj];
					}

					params->XORENC++;
				}
			}

			BitPos++;
			TmpM[i] /= 2.0;
			if (TmpM[i] == 0)
				break;
		}
	}

	char **cc = MallocCharTwoArray(params->L, params->SYMBOL_SIZE);
	for (i = 0; i < params->L; i++)
		memcpy(cc[i], c[i], params->SYMBOL_SIZE);

	//LT coding
	char **e = LTCode(cc, M, params, 0);

	free(TmpM);
	FreeCharTwoArray(c, params->L);
	FreeCharTwoArray(cc, params->L);

	return e;
}


char** RaptorDecoder(char **data, int dataMap[], raptor_params_t *params)
{
	int K = params->K;
	int S = params->S;
	int N = params->N;
	int L = params->L;
	int i, j, jj, a, b;
	int count, aaa, BitPos;
	unsigned int *TmpM;
	char **c;

	params->XORDEC = 0;

	char **Anew = FindMatrixAUsingSymbols(dataMap, params);
	InverseHALF(Anew, N, params);
	InverseLDPC(Anew, N, params);
	c = IntermediateKSymbols(Anew, data, params);

	if (c == NULL)
	{
		printf("RaptorDecoder Failed!! (@IntermediateKSymbols)\n");
		FreeCharTwoArray(Anew, N);
		return NULL;
	}


	//LDPC
	for (i = 0; i < K; i++)
	{
		a = 1 + ((int)(floor((double)(i/S)))%(S-1));
		b = i % S;

		// yunmin - AVX
		if (VECTOR)
		{
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
			b = (b+a)%S;
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
			b = (b+a)%S;
			vector_xor(c[K+b], c[K+b], c[i], params->SYMBOL_SIZE);
		}
		else
		{
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
			b = (b+a)%S;
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
			b = (b+a)%S;
			for (j = 0; j < params->SYMBOL_SIZE; j++)
				c[K+b][j] ^= c[i][j];
		}

		params->XORDEC += 3;
	}

	//HALF symbol
	TmpM = (unsigned int*) malloc(sizeof(unsigned int)*(K + S));
	memcpy(TmpM, params->m, sizeof(unsigned int)*(K + S));

	for (i = 0; i < K+S; i++)
	{
		count = TmpM[i];
		BitPos = 0;

		for (j = 0; ; j++)
		{
			aaa = TmpM[i] % 2;

			if (aaa == 1)
			{
				if (BitPos == j)
				{
					// yunmin - AVX
					if (VECTOR)
					{
						vector_xor(c[K+S+j], c[K+S+j], c[i], params->SYMBOL_SIZE);
					}
					else
					{
						for (jj = 0; jj < params->SYMBOL_SIZE; jj++)
							c[K+S+j][jj] ^= c[i][jj];
					}

					params->XORDEC++;
				}
			}

			BitPos++;
			TmpM[i] /= 2.0;
			if(TmpM[i] == 0) break;
		}
	}

	char **cc = MallocCharTwoArray(params->L, params->SYMBOL_SIZE);
	for (i = 0; i<params->L; i++)
		memcpy(cc[i], c[i], params->SYMBOL_SIZE);

	//LT coding
	char **d = LTCode(cc, K, params, 1);

	free(TmpM);
	FreeCharTwoArray(Anew, N);
	FreeCharTwoArray(c, L);
	FreeCharTwoArray(cc, L);

	return d;
}


// Raptor encoding process
char *raptor_encoding(raptor_params_t *params, char *src_data, int sym_size, int num_src_syms)
{
	int i,j ;
	int num_enc_syms;
	int num_red_syms;
	char *red_data;
	char **src_block;
	char **enc_block;
	//raptor_params_t *params;

	//params = get_raptor_params(raptor_params, sym_size, num_src_syms);
	src_block = MallocCharTwoArray(num_src_syms, sym_size);
	num_enc_syms = (int) ceil((double)num_src_syms / 0.8); //RAPTOR_CODE_RATE);
	num_red_syms = num_enc_syms - num_src_syms;
	red_data = (char *) malloc(sym_size * num_red_syms);

	// Convert src_data(1D array) to src_block(2D array)
	for (i = 0; i < num_src_syms; i++)
		memcpy(src_block[i], src_data+(i*sym_size), sym_size);

	// Raptor encoding
	enc_block = RaptorEncoder2(src_block, params, num_enc_syms);
	if (enc_block == NULL)
	{
		return NULL;
	}

	// Convert enc_block(2D array) to red_data(1D array)
	for (i = 0; i < num_red_syms; i++)
		memcpy(red_data+(i*sym_size), enc_block[num_src_syms+i], sym_size);

	FreeCharTwoArray(src_block, num_src_syms);
	FreeCharTwoArray(enc_block, num_enc_syms);

	return red_data;
}

// Raptor decoding process
char *raptor_decoding2(raptor_params_t *params, char *enc_data, int *symbol_map, int sym_size, int num_src_syms)
{
	char **enc_block = NULL;
	char *dec_block_1d = NULL;
	char **dec_block_2d = NULL;

	// printf("Symbol Map print %d \n", offset++);
	// for (int i = 0; i < params->N; i++)
	// {
	// 	printf("symbol_map[%d]=%d \n", i, symbol_map[i]);
	// }
	// fflush(stdout);

	enc_block = MallocCharTwoArray(160, sym_size);
	dec_block_1d = (char *) malloc(sizeof(char) * (sym_size * num_src_syms));

	// Convert enc_data(1D array) to enc_block(2D array)
	for (int i = 0; i < params->N; i++)
	{
		memcpy(enc_block[i], enc_data + (i*sym_size), sym_size);
	}

	// Raptor decode
	dec_block_2d = RaptorDecoder(enc_block, symbol_map, params);

	if (dec_block_2d == NULL)
	{
		printf("Raptor decoding failure!\n");
		return NULL;
	}

	// Convert dec_block_2d(2D array) to dec_block_1d(1D array)
	for (int i = 0; i < num_src_syms; i++)
	{
		memcpy(dec_block_1d + (i*sym_size), &dec_block_2d[i][0], sym_size);
	}

	// Print dec_block
	//print_array(dec_block_1d, sym_size * num_src_syms);

	return dec_block_1d;
}


void vector_xor(char *dst, char *src1, char *src2, int iter_num)
{
	if (BITS16)
	{
		int i = 0;
		__m128i a, b;
		__m128i result;
		for (i = 0; i < iter_num; i+=16)
		{
			a = _mm_loadu_si128((__m128i const*)(src1+i));
			b = _mm_loadu_si128((__m128i const*)(src2+i));
			result = _mm_xor_si128(a, b);
			_mm_storeu_si128((__m128i_u *)(dst+i), result);
		}
	}
	else
	{
		// int i = 0;
		// __m256 a, b;
		// __m256 result;
		// for (i = 0; i < iter_num; i+=32)
		// {
		// 	a = _mm256_loadu_ps((const float *)(src1+i));
		// 	b = _mm256_loadu_ps((const float *)(src2+i));
		// 	result = _mm256_xor_ps(a, b);
		// 	_mm256_storeu_ps((float*)dst+i, result);
		// }
	}
}

raptor_params_t *get_raptor_params(raptor_params_t *params[], int symbol_size, int num_symbols)
{
	int i;
	for (i = 0; i < 1; i++)
		if (params[i]->SYMBOL_SIZE == symbol_size && params[i]->K == num_symbols)
			return params[i];

	printf("Can't find index of raptor parameters (s=%d, k=%d) \n", symbol_size, num_symbols);

	return NULL;
}

void print_array(char *block, int length)
{
	for (int i = 0; i < length; i++)
	{
		if (i % 32 == 0)
			printf("\n%6d: ", i);
		printf("%4d ", (int)block[i]);
	}
	fflush(stdout);
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/lucas-clemente/quic-go/internal/protocol"
)

var SymbolSize = []uint32{1024}
var NumberOfSrcSymbolSize = []uint32{16} //128, 192, 256, 320, 384, 448, 512 }
var MinimumOverhead = []float32{0.0758, 0.0518, 0.0358, 0.0310, 0.0260, 0.0224, 0.0187}
var CodeRate = 0.8

type Raptor struct {
	raptor_params [1](*C.raptor_params_t)
}

func (r *Raptor) RaptorInit() {
	//FecLog("raptor.go: RaptorInit()")
	for i := 0; i < len(SymbolSize); i++ {
		s := SymbolSize[i]
		k := NumberOfSrcSymbolSize[0]

		// K, S, min overhead, overhead (int K, int SYMBOL_SIZE, double minCR, double CR)
		r.raptor_params[i] = C.RaptorInitialization(C.int(k), C.int(s), C.double(MinimumOverhead[0]), C.double(CodeRate))
		if r.raptor_params[i] == nil {
			err := fmt.Sprintf("Raptor initialization failed!! (s=%d, k=%d) \n", s, k)
			panic(err)
		} else {
			//FecLog("raptor.go: RaptorInit(): Raptor Parameter Set Initialization...(%d/%d)       \n", i, SymbolSize[i])
		}
	}
}

func (r *Raptor) RaptorEncode(data []byte, symSize uint32, numSrcSyms uint32) []byte {
	srcData := (*C.char)(unsafe.Pointer(&data[0]))

	// Raptor encoding - it reurns 1-D redundant data array only
	fecBlock := C.raptor_encoding((*C.raptor_params_t)(r.raptor_params[0]), (*C.char)(srcData), C.int(symSize), C.int(numSrcSyms))

	if fecBlock == nil {
		FecLog("raptor.go: RaptorEncode(): Raptor encoding failed!!!\n")
		return nil
	}

	return C.GoBytes(unsafe.Pointer(fecBlock), C.int(SymbolSize[0]*4)) // 4, 32
}

func (r *Raptor) RaptorDecode(offset protocol.ByteCount, data []byte, symbolMap []uint32, symSize uint32, numSrcSyms uint32) []byte {
	encBlock := (*C.char)(unsafe.Pointer(&data[0]))
	symMap := (*C.int)(unsafe.Pointer(&symbolMap[0]))

	decBlock := C.raptor_decoding2((*C.raptor_params_t)(r.raptor_params[0]), (*C.char)(encBlock), (*C.int)(symMap), C.int(symSize), C.int(numSrcSyms))
	if decBlock == nil {
		return nil
	}

	return C.GoBytes(unsafe.Pointer(decBlock), C.int(SymbolSize[0]*NumberOfSrcSymbolSize[0]))
}
